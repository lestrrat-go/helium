<xsl:transform
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:ged="http://www.wrox.com/569090/gedcom"
   xmlns="http://www.w3.org/1999/xhtml"
   exclude-result-prefixes="xs ged"                          
   version="2.0" >
   
   <!-- A "system test" of schema-aware processing -->
   
   <?spec xslt#stylesheets-and-schemas?>
<xsl:strip-space elements="*"/>   

<!-- import the schema for the GEDCOM 6.0 vocabulary -->

<xsl:import-schema schema-location="gedSchema.xsd"/>

<!-- import the schema for the target XHTML vocabulary -->

<!--<xsl:import-schema namespace="http://www.w3.org/1999/xhtml"
                   schema-location="http://www.w3.org/2002/08/xhtml/xhtml1-transitional.xsd"/>-->
                   
<xsl:output method="xhtml" indent="yes" encoding="iso-8859-1"/>                   

<!-- parameter to supply the Id of the person to be displayed. 
     Default value is the Id of the first person in the data set -->
  
<xsl:param name="id" select="/*/IndividualRec[1]/@Id" as="xs:string"/>

<!-- keys to allow records to be found by their Id -->

<xsl:key name="indi" match="IndividualRec" use="@Id"/>
<xsl:key name="fam" match="FamilyRec" use="@Id"/>

<!-- a key that locates the family record for a given child -->

<xsl:key name="family-of-child" match="FamilyRec" use="Child/Link/@Ref"/>

<!-- a key that locates the family records for a given parent -->

<xsl:key name="families-of-parent" match="FamilyRec" use="element(*,ParentType)/Link/@Ref"/>

<!-- a key to allow events to be found for a given individual -->

<xsl:key name="events-for-person" match="EventRec" use="Participant/Link/@Ref"/>

<!-- a function to get all the events for a given individual -->

<xsl:function name="ged:events-for-person" as="element(EventRec)*">
  <xsl:param name="person" as="element(IndividualRec)"/>
  <xsl:sequence select="$person/key('events-for-person', $person/@Id)"/>
</xsl:function>

<!-- a function to get all the families in which a given individual is a spouse -->

<xsl:function name="ged:families-of-spouse" as="element(FamilyRec)*">
  <xsl:param name="person" as="element(IndividualRec)"/>
  <xsl:sequence select="$person/key('families-of-parent', $person/@Id)"/>
</xsl:function>

<!-- function to get all the events for a couple -->

<xsl:function name="ged:events-for-couple" as="element(EventRec)*">
  <xsl:param name="couple" as="element(FamilyRec)"/>
  <xsl:sequence
    select="if ($couple/HusbFath and $couple/WifeMoth)
            then (ged:events-for-person($couple/key('indi', $couple/HusbFath/Link/@Ref))
                    intersect
                  ged:events-for-person($couple/key('indi', $couple/WifeMoth/Link/@Ref)))
            else ()"/>

</xsl:function>

<!-- function to get the marriage event for a couple --> 

<xsl:function name="ged:marriage-event" as="element(EventRec)?">
  <xsl:param name="couple" as="element(FamilyRec)"/>
  <xsl:variable name="marriage-vitals"
                as="element(EventRec)*" 
                select="ged:events-for-couple($couple)[@VitalType='marriage']"/>
  <xsl:variable name="marriage"
                as="element(EventRec)*"  
                select="$marriage-vitals[@Type='marriage']"/>          
  <xsl:sequence 
    select="if ($marriage) 
               then $marriage[1]
            else if ($marriage-vitals)
               then $marriage-vitals[1]
            else ()"/> 
</xsl:function>

<!-- function to get the birth date of an individual -->

<xsl:function name="ged:birth-date" as="element(*,DateType)">
  <xsl:param name="person" as="element(IndividualRec)"/>
  <xsl:variable name="birth-events"
    select="ged:events-for-person($person)[@Type='birth']"/>
  <xsl:sequence
    select="if (exists($birth-events/Date))
              then ($birth-events/Date)[1]
              else (ged:events-for-person($person)[@VitalType='birth']/Date)[1]"/>
</xsl:function>

<!-- function to get the estimated marriage date for a couple -->

<xsl:function name="ged:estimated-marriage-date" as="element(*,DateType)">
  <xsl:param name="couple" as="element(FamilyRec)"/>
  <xsl:for-each select="$couple">
    <xsl:variable name="marriage-vitals" 
      select="if (HusbFath and WifeMoth)
              then (ged:events-for-person(key('indi', HusbFath/Link/@Ref))
                    intersect
                  ged:events-for-person(key('indi', WifeMoth/Link/@Ref)))
                  [@VitalType='marriage']
              else ()"/>
    <xsl:variable name=   "marriage"
                  as=     "element(EventRec)*" 
                  select= "$marriage-vitals[@Type='marriage']"/>
                          
    <xsl:variable name=   "marriage-date"
                  as=     "element(*, DateType)?" 
                  select= "if ($marriage/Date) 
                             then ($marriage/Date)[1]
                           else if ($marriage-vitals/Date)
                             then ($marriage-vitals/Date)[1]
                           else ()"/>
                         
    <xsl:choose>
      <xsl:when test="$marriage-date">
        <xsl:sequence select="$marriage-date"/>
      </xsl:when>
      <xsl:otherwise>
        <xsl:variable name=   "childbirth-dates" 
                      as=     "element(*, DateType)*"
                      select= "Child/Link/@Ref/key('indi',.)/ged:birth-date(.)"/>
                
        <xsl:for-each select="$childbirth-dates">
          <xsl:sort select="ged:date-sort-key(.)"/>
          <xsl:if test="position() eq 1"><xsl:sequence select="."/></xsl:if>
        </xsl:for-each>
      </xsl:otherwise>
    </xsl:choose>
  </xsl:for-each>
</xsl:function> 

<!-- function to convert a standard GEDCOM date (DD MMM YYYY) to an xs:date -->

<xsl:function name="ged:date-to-ISO" as="xs:date">
  <xsl:param name="date" as="StandardDate"/>
  <xsl:variable name="iso-date">
    <xsl:analyze-string select="$date" regex="^\s*([0-9]+)\s+([A-Z]+)\s+([0-9]+)\s*$">
      <xsl:matching-substring>
        <xsl:number value="regex-group(3)" format="0001"/>
        <xsl:text>-</xsl:text>
        <xsl:number value="index-of(('JAN', 'FEB', 'MAR', 'APR', 'MAY', 'JUN',
                                     'JUL', 'AUG', 'SEP', 'OCT', 'NOV', 'DEC'),
                                    regex-group(2))" format="01"/>
        <xsl:text>-</xsl:text>
        <xsl:number value="regex-group(1)" format="01"/>
      </xsl:matching-substring>
    </xsl:analyze-string>
  </xsl:variable>
  <xsl:sequence select="xs:date($iso-date)"/>
</xsl:function>

<!-- function to format a standard GEDCOM date for display -->

<xsl:function name="ged:format-date" as="xs:string">
  <xsl:param name="date" as="StandardDate"/>
  <xsl:sequence select="format-date(ged:date-to-ISO($date), '[D] [MNn] [Y]')"/>
</xsl:function>                           


<xsl:function name="ged:date-sort-key" as="xs:string">
  <xsl:param name="date" as="element(*,DateType)"/>
  <xsl:sequence select="
        if (data($date) instance of StandardDate)
          then string(ged:date-to-ISO($date))
          else substring($date, string-length($date)-4)
        "/>
</xsl:function>      

<!-- function to get the year from a GEDCOM date -->

<xsl:function name="ged:get-year" as="xs:string">
  <xsl:param name="date" as="element(*,DateType)"/>
  <xsl:sequence select="substring($date, string-length($date)-4)"/>
</xsl:function>

<!-- a function to capitalize the initial letter of a string -->

<xsl:function name="ged:initial-cap" as="xs:string">
  <xsl:param name="input" as="xs:string"/>
  <xsl:sequence select="concat(upper-case(substring($input, 1, 1)),
                               substring($input, 2))"/>
</xsl:function>  
   
                    

<xsl:template match="/">
  <xsl:if test="not(/* instance of element(GEDCOM))">
    <xsl:message terminate="yes">Input document is not a validated GEDCOM 6.0 file</xsl:message>
  </xsl:if>
  <xsl:result-document validation="lax">
    <xsl:variable name="person" select="key('indi', $id)"/>
    <xsl:apply-templates select="$person"/>
  </xsl:result-document>
</xsl:template>

<xsl:template match="IndividualRec">
	<html>
        <head>
            <xsl:call-template name="css-style"/>
            <xsl:variable name="name">
                <xsl:apply-templates select="IndivName[1]"/>
            </xsl:variable>
            <title><xsl:value-of select="$name"/></title>
        </head>

        <body bgcolor="{if (Gender='M') then 'cyan' else 'pink'}">

            <!-- Show name and parentage -->

            <h1><xsl:apply-templates select="IndivName[1]"/></h1>
            <xsl:if test="IndivName[2]">
              <p>
                <span class="label">Also known as: </span>
                <xsl:for-each select="IndivName[position() ge 2]">
                  <xsl:apply-templates select="."/>
                  <xsl:if test="position() ne last()">, </xsl:if>
                </xsl:for-each>
              </p>
            </xsl:if>
            <xsl:call-template name="show-parents"/>
            <hr/>

            <table>
            <tr>

                <!-- Show events and attributes -->

                <td width="50%" valign="top">
                    <xsl:call-template name="show-events"/>                        
                </td>
                <td width="20%"/>

                <!-- Show children -->

                <td width="30%" valign="top">
                    <xsl:call-template name="show-partners"/>
                </td>
            </tr>
            </table>

            <hr/>

            <!-- Show notes -->

            <xsl:for-each select="Note">
                <p class="text"><xsl:apply-templates mode="note"/></p>
                <xsl:if test="position() eq last()"><hr/></xsl:if>
            </xsl:for-each>

        </body>
    </html>
</xsl:template>


   
<xsl:template name="show-parents">

  <xsl:variable name=   "parental-family"
                as=     "element(FamilyRec)?"
                select= "key('family-of-child', @Id)[1]"/>
                  
  <xsl:variable name=   "father"
                as=     "element(IndividualRec)?" 
                select= "key('indi', $parental-family/HusbFath/Link/@Ref)"/>
                  
  <xsl:variable name=   "mother"
                as=     "element(IndividualRec)?"   
                select= "key('indi', $parental-family/WifeMoth/Link/@Ref)"/>

  <p>
    <xsl:if test="$father">
        <span class="label">Father: </span><xsl:apply-templates select="$father/IndivName" mode="link"/>&#xa0;
    </xsl:if>
    <xsl:if test="$mother">
        <span class="label">Mother: </span><xsl:apply-templates select="$mother/IndivName" mode="link"/>&#xa0; 
    </xsl:if>
  </p>
</xsl:template>

<!-- Show the events for an individual -->

<xsl:template name="show-events">
  <xsl:variable name=   "subject"
                as=     "element(IndividualRec)"
                select= "."/>
                
  <xsl:for-each select="ged:events-for-person(.)">
        <xsl:sort select="ged:date-sort-key(Date)"/>

        <h3><xsl:value-of select="ged:initial-cap(@Type)"/></h3>
        <p>
        <xsl:for-each select="Participant[Link/@Ref ne $subject/@Id]">
            <span class="label"><xsl:value-of select="ged:initial-cap(Role)"/>: </span>
            <xsl:apply-templates select="Link/@Ref/key('indi',.)/IndivName[1]" mode="link"/>
            <br/>
        </xsl:for-each>
        <xsl:if test="Date">
            <span class="label">Date: </span><xsl:apply-templates select="Date"/><br/>
        </xsl:if>
        <xsl:if test="Place">
            <span class="label">Place: </span><xsl:apply-templates select="Place"/><br/>
        </xsl:if>
        </p>
        <xsl:for-each select="Note">
            <p class="text"><xsl:apply-templates mode="note"/></p>
        </xsl:for-each>
    
    </xsl:for-each>
</xsl:template>

<xsl:template name="show-partners">
  <xsl:variable name=   "subject" 
                as=     "element(IndividualRec)"
                select= "."/>
                
  <xsl:variable name=   "partnerships"
                as=     "element(FamilyRec)*" 
                select= "ged:families-of-spouse(.)"/>
                
  <xsl:for-each select="$partnerships">
    <xsl:sort select="ged:date-sort-key(ged:estimated-marriage-date(.))"/>
        
    <xsl:variable name=   "partner"
                  as=     "element(IndividualRec)?" 
                  select= "key('indi', element(*, ParentType)/Link/@Ref) except $subject"/>

    <xsl:variable name=   "partner-seq"
                  as=     "xs:integer?"
                  select= "if(count($partnerships) eq 1)
                           then ()
                           else position()"/>

    <xsl:if test="$partner">
      <h2>Partner <xsl:value-of select="$partner-seq"/></h2>
      <p><xsl:apply-templates select="$partner/IndivName[1]" mode="link"/></p>
    </xsl:if>

    <xsl:if test="Child">
      <h3>Children:</h3>
      <p>
        <xsl:for-each select="Child">
          <xsl:sort select="ChildNbr"/>
          <xsl:sort select="ged:date-sort-key(Link/@Ref/key('indi',.)/ged:birth-date(.))"/>
          
          <xsl:variable name=   "child"
                        as=     "element(IndividualRec)"
                        select= "Link/@Ref/key('indi',.)"/>
                        
          <xsl:value-of select="ged:get-year(ged:birth-date($child))"/>
          <xsl:text> </xsl:text>
          <xsl:apply-templates select="$child/IndivName[1]" mode="link"/><br/>
        </xsl:for-each>
      </p>
    </xsl:if>
  </xsl:for-each>
</xsl:template>

<xsl:template match="IndivName" mode="link">
	<a>
		<xsl:attribute name="href">
			<xsl:call-template name="make-href"/>
		</xsl:attribute>
		<xsl:apply-templates/>
	</a>
</xsl:template>

<xsl:template match="NamePart[@Type='surname']">
    <xsl:text> </xsl:text>
    <span class="surname"><xsl:apply-templates/></span>
    <xsl:text> </xsl:text>
</xsl:template>

<xsl:template name="make-href">
	<xsl:value-of select="concat(../@Id, '.html')"/>
</xsl:template>

<xsl:template match="PlaceName[PlacePart]">
  <xsl:variable name="sorted-parts" as="element()*">
    <xsl:perform-sort select="PlacePart">
      <xsl:sort select="@Level" order="descending"/>
    </xsl:perform-sort>
  </xsl:variable>
  <xsl:value-of select="$sorted-parts" separator=", "/>
</xsl:template>

<xsl:template match="Date[data(.) instance of StandardDate]">
  <xsl:value-of select="ged:format-date(data(.))"/>
</xsl:template>

<xsl:template match="Date">
  <xsl:value-of select="."/>
</xsl:template>

<xsl:template match="text()" mode="note">
  <xsl:analyze-string select="." regex="\n">
    <xsl:matching-substring>
      <br/>
    </xsl:matching-substring> 
    <xsl:non-matching-substring>
      <xsl:value-of select="."/>
    </xsl:non-matching-substring>  
  </xsl:analyze-string>
</xsl:template>   

<xsl:template name="css-style">
    <style type="text/css">

	H1 {
	    font-family: Verdana, Helvetica, sans-serif;
	    font-size: 18pt;
	    font-weight: bold;
	    color: "#FF0080"
	}

	H2 {
	    font-family: Verdana, Helvetica, sans-serif;
	    font-size: 14pt;
	    font-weight: bold;
	    color: black;
	}

	H3 {
	    font-family: Lucida Sans, Helvetica, sans-serif;
	    font-size: 11pt;
	    font-weight: bold;
	    color: black;
	}

	SPAN.surname {
	    font-weight: bold;
        text-decoration: underline;
	}

	SPAN.label {
	    font-family: Lucida Sans, Helvetica, sans-serif;
	    font-size: 10pt;
	    font-weight: normal;
        font-style: italic;
	    color: black;
	}

    P,LI,TD {
	    font-family: Lucida Sans, Helvetica, sans-serif;
	    font-size: 10pt;
	    font-weight: normal;
	    color: black;       
	}

    P.text {
	    font-family: Comic Sans MS, Helvetica, sans-serif;
	    font-size: 10pt;
	    font-weight: normal;
	    color: black;       
	}

    </style>
</xsl:template>

</xsl:transform>
