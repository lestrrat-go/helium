<xsl:stylesheet version="2.0" 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- Saxon bug 3479632	xsl:template match="node()" mode="#all", reported by Gerrit Imsieke -->

   <xsl:template match="node()" mode="#all">
     <xsl:copy>
       <xsl:apply-templates mode="#current"/>
     </xsl:copy>
   </xsl:template>
   
   <xsl:template match="foo" mode="baz">
     <bar>
       <xsl:apply-templates mode="#current"/>
     </bar>
   </xsl:template>
   
</xsl:stylesheet>