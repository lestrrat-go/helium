<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                 <!--It is a dynamic error
                     for a stylesheet to write to an external
                     resource and read from the same resource during a single transformation,
                     if the same absolute URI is used to access the resource in both cases.                        
                  -->
                  
   <xsl:template name="main">
      <out>
         <xsl:variable name="a" select="doc('error-1500a.xml')"/>
         <xsl:copy-of select="$a"/>
         <xsl:result-document href="{document-uri($a)}">
           <boo/>
         </xsl:result-document>
      </out>
   </xsl:template>
</xsl:stylesheet>
