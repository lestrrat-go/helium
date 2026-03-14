<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-substring-before?>
  <!-- Purpose: Test of 'substring-before()' function with 'substring-before()' 
       as an argument.  The second argument of the outer 'substring-before()'
       function is build by calling another 'substring-before()' function.-->

<xsl:template match="doc">
 <out>
   <xsl:value-of select="substring-before('Correct Substring Beforewrong',substring-before('wrongnogood','nogood'))"/>
 </out>
</xsl:template>
</xsl:stylesheet>