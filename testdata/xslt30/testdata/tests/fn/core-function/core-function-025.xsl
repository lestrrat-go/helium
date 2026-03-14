<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-substring-before?>
  <!-- Purpose: Test of 'substring-before()' function with a variable
       reference as the argument.  Both arguments of the function are
       build by referencing a variable. -->

<xsl:variable name="variable1">substring before with variablecut this</xsl:variable>
<xsl:variable name="variable2">cut this</xsl:variable>

<xsl:template match="doc">
 <out>
   <xsl:value-of select="substring-before($variable1,$variable2)"/>
 </out>
</xsl:template>
</xsl:stylesheet>