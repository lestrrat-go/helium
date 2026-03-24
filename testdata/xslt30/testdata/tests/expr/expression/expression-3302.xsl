<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#combining_seq?>
  <!-- Purpose: test union operator at different nesting levels (different elements) -->
  <!--  -->

 <xsl:template match = "doc">
   <out><xsl:text>
   </xsl:text>
     <xsl:for-each select = "book">
       <xsl:apply-templates select= "author/name|author/bibliography/author/chapters"/>
     </xsl:for-each>
   </out>
 </xsl:template>
 
 <xsl:template match="*">
   <xsl:value-of select="."/><xsl:text>
   </xsl:text>
 </xsl:template>
</xsl:stylesheet>