<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
                 <!--It is a static error if the
                     select attribute of xsl:sequence is present and the
                     instruction has children other than xsl:fallback.-->
                     
   <xsl:template name="main">
      <out>
         <xsl:sequence select="1 to 10">
           <xsl:value-of select="5"/>
         </xsl:sequence>
      </out>
   </xsl:template>
</xsl:stylesheet>
