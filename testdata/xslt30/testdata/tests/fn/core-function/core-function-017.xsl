<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-substring-before?>
  <!-- Purpose: Test of 'substring-before()' function with 'substring-before()' 
       as an argument.  Both arguments in the outer 'substring-before()'
       function are build by invoking corresponding 'substring-before()' functions.-->

<xsl:template match="doc">
 <out>
   <xsl:value-of select="substring-before(substring-before('Correct Substring Beforewrongcut here','cut here'),substring-before('wrongnogood','nogood'))"/>
 </out>
</xsl:template>
</xsl:stylesheet>