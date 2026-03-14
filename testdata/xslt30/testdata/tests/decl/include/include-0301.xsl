<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

   <?spec xslt#apply-imports?>
   <!-- Test more that one call of xsl:apply-imports from the same template -->
   <!-- See Saxon bug 6.2.2/019 -->

   <xsl:import href="include-0301a.xsl"/>

   <xsl:template match="doc">
      <bat>
         <baz id="1">
            <xsl:apply-imports/>
         </baz>
         <baz id="2">
            <xsl:apply-imports/>
         </baz>
         <baz id="3">
            <xsl:apply-imports/>
         </baz>
      </bat>
   </xsl:template>

</xsl:stylesheet>
