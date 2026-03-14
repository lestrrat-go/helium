<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">


  <!-- PURPOSE: test that the statically declared param and variable (with static="1") are allowed in an excluded section. -->
  <?spec xslt#conditional-inclusion?>
  <xsl:variable name="xx" select="2" static=" 1 "/> 
  <xsl:param name="xxx" select="3" static="1" /> 
  <xsl:template match="*" use-when="some $x in (1,2,3) satisfies $x=$xxx">
    <p>value matched ok $xx=<xsl:value-of select="$xx" /></p>
  </xsl:template>

</xsl:stylesheet>