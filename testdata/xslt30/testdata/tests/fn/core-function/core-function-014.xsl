<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-substring-after?>
  <!-- Purpose: Test of 'substring-after()' function with 'substring-after()' 
       as an argument.  The second argument of the outer function is
       build by calling another "substring-after()' function.-->

<xsl:template match="doc">
 <out>
   <xsl:value-of select="substring-after('wrongCorrect Substring After',substring-after('nogoodstringwrong','nogoodstring'))"/>
 </out>
</xsl:template>
</xsl:stylesheet>