<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <!-- Test that xsl:strip-space applies to result of document() function -->
   
   <xsl:strip-space elements="a b"/>

   <xsl:template match="/">
      <out>
         <xsl:copy-of select="document('doc14.xml')"/>
      </out>
   </xsl:template>

</xsl:stylesheet>
