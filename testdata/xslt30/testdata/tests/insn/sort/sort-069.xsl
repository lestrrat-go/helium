<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Use sequence constructor in xsl:sort and refer to last() -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:template match="/">
      <out>List of numbers: <xslt:value-of select="doc/a"/>
         <Last_BeforeSort>
            <xslt:value-of select="doc/a[position()=last()]"/>
         </Last_BeforeSort>
         <xslt:for-each select="doc/a">
        	   <xslt:sort data-type="number" order="descending">
        		     <xslt:call-template name="temp"/>
        	   </xslt:sort>
        	   <xslt:if test="position()=last()">
        	      <last_AfterSort>
                  <xslt:value-of select=".[position()=last()]"/>
               </last_AfterSort>
        	   </xslt:if>
        </xslt:for-each>
      </out>                                       
  </xslt:template>

   <xslt:template match="text()"/>

   <xslt:template name="temp">
	     <xslt:value-of select="xs:string(.)"/>
   </xslt:template>
</xslt:transform>
