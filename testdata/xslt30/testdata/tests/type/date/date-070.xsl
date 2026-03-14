<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" 
xmlns:xdt="http://www.w3.org/2005/04/xpath-datatypes" exclude-result-prefixes="xs xdt">

<xsl:output encoding="iso-8859-1"/>

<?spec fo#func-add-dayTimeDuration-to-dateTime?>
<?spec fo#func-add-yearMonthDuration-to-dateTime?>

  <!-- PURPOSE: XSLT 2.0: test dateTime +/- duration -->

<xsl:param name="date" select="xs:dateTime('2004-02-29T00:12:00Z')"/>
<xsl:param name="d5" select="xs:dayTimeDuration('P5DT4H0M0.5S')"/>
<xsl:param name="d-5" select="xs:dayTimeDuration('-P5DT4H0M0.5S')"/>
<xsl:param name="m2" select="xs:yearMonthDuration('P2M')"/>
<xsl:param name="m-2" select="xs:yearMonthDuration('-P2M')"/>
<xsl:param name="y2" select="xs:yearMonthDuration('P2Y')"/>
<xsl:param name="y-4" select="xs:yearMonthDuration('-P4Y')"/>

  <xsl:template match="/">
     <out>
       <a><xsl:value-of select="$date + $d5"/></a>
       <a><xsl:value-of select="$date + $d-5"/></a>
       <a><xsl:value-of select="$date + $m2"/></a>
       <a><xsl:value-of select="$date + $m-2"/></a>
       <a><xsl:value-of select="$date + $y2"/></a>
       <a><xsl:value-of select="$date + $y-4"/></a> 
                                         
       <b><xsl:value-of select="$date - $d5"/></b>
       <b><xsl:value-of select="$date - $d-5"/></b>
       <b><xsl:value-of select="$date - $m2"/></b>
       <b><xsl:value-of select="$date - $m-2"/></b>
       <b><xsl:value-of select="$date - $y2"/></b>
       <b><xsl:value-of select="$date - $y-4"/></b>   

       <a><xsl:value-of select="$d5 + $date"/></a>
       <a><xsl:value-of select="$d-5 + $date"/></a>
       <a><xsl:value-of select="$m2 + $date"/></a>
       <a><xsl:value-of select="$m-2 + $date"/></a>
       <a><xsl:value-of select="$y2 + $date"/></a>
       <a><xsl:value-of select="$y-4 + $date"/></a>                      
     </out>     
  </xsl:template>

</xsl:stylesheet>
