<?xml version="1.0" standalone='no'?>
<xsl:stylesheet
		version="2.0"
		xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
		xmlns="http://www.w3.org/TR/REC-html40"
  >

<?spec xpath#static_context?>
<!-- PURPOSE: test that the default namespace is not applied to names in XPath expressions
     or to variable names -->


 <xsl:template match="/">
    <out>
    Correct output: 8 lines, saying "value is n" where n is 1..8 
   <xsl:for-each select="data/row">
     value is <xsl:value-of select="."/>
   </xsl:for-each>
   <xsl:for-each select="$temp1">
     value is <xsl:value-of select="4+."/>
   </xsl:for-each>
   </out>
  </xsl:template>
 <xsl:variable name="temp1" select="data/row"/>
  
</xsl:stylesheet>


