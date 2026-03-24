<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"><!--It is a static error if the
                        select attribute of the xsl:map-entry element
                        is present unless the element has has no children other than xsl:fallback elements.-->
   <xsl:template name="main">
      <out>
         <xsl:variable name="m" as="map(*)">
           <xsl:map>
             <xsl:map-entry key="1" select="22">
               <xsl:sequence select="23"/>
             </xsl:map-entry>
           </xsl:map>
         </xsl:variable>
         <xsl:value-of select="map:size($m)" xmlns:map="http://www.w3.org/2005/xpath-functions/map"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
