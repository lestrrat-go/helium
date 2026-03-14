<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#combining_seq?>
  <!-- Purpose: NodeSet union using a function that return a node set (id). and 
       a location path axes. -->
  <!--  -->

 <xsl:key name="key1" match="book" use="@author"></xsl:key>

 <xsl:template match="doc">
   <out><xsl:text>
   </xsl:text>
      <xsl:apply-templates select="key('key1','Carmelo Montanez')|child::article"/>
   </out>
 </xsl:template>

 <xsl:template match = "*">
   <xsl:value-of select = "."/><xsl:text>
   </xsl:text>
 </xsl:template>
</xsl:stylesheet>