<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#combining_seq?>
  <!-- Purpose: NodeSet union using the ancestor location path-->
  <!--  -->

 <xsl:template match="/">
   <out>
   <xsl:for-each select="//child1|//child2">
     <xsl:apply-templates select="ancestor::sub1|ancestor::sub2"/>
   </xsl:for-each>
  </out>
 </xsl:template>

 <xsl:template match="*">
   <xsl:value-of select="."/>
 </xsl:template>
</xsl:stylesheet>

