<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-concat?>
  <!-- Purpose: Test of 'concat()' function with a variable reference 
       as the argument.  All arguments of the function are
       build by referencing a variable. -->

<xsl:variable name="variable1">String </xsl:variable>
<xsl:variable name="variable2">From </xsl:variable>
<xsl:variable name="variable3">Variable</xsl:variable>

<xsl:template match="doc">
 <out>
   <xsl:value-of select="concat($variable1,$variable2,$variable3)"/>
 </out>
</xsl:template>
</xsl:stylesheet>