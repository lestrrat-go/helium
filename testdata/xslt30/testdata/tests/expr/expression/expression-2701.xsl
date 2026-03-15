<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#combining_seq?>
  <!-- Purpose: NodeSet union using the descendant location path-->
  <!--  -->

 <xsl:template match="/">
   <out><xsl:text>
   </xsl:text>
     <xsl:for-each select="doc">
       <xsl:apply-templates select="descendant::child1|descendant::child2"/>
     </xsl:for-each>
   </out>
 </xsl:template>

 <xsl:template match="*">
   <xsl:value-of select="."/><xsl:text>
   </xsl:text>
 </xsl:template>
</xsl:stylesheet>