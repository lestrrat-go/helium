<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#combining_seq?>
  <!-- Purpose: test union operator using a variable and a location path. -->
  <!--  -->

 <xsl:template match = "doc">
   <xsl:variable name = "var1" select = "//child1"></xsl:variable>
   <out><xsl:text>
   </xsl:text>
     <xsl:apply-templates select = "$var1|child::child2"/>
   </out>
 </xsl:template>

 <xsl:template match = "*">
   <xsl:value-of select = "."/><xsl:text>
   </xsl:text>
 </xsl:template>
</xsl:stylesheet>