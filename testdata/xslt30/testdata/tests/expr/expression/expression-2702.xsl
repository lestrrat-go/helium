<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#combining_seq?>
  <!-- Purpose: NodeSet union using the descendant-or-self location path-->
  <!--  -->

 <xsl:template match="/">
   <out>
     <xsl:for-each select="doc">
       <xsl:apply-templates select="descendant-or-self::doc|descendant-or-self::doc"/>
     </xsl:for-each>
   </out>
 </xsl:template>

 <xsl:template match="*">
   <xsl:value-of select = "."/>
 </xsl:template>
</xsl:stylesheet>

