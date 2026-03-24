<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- CaseName: numbering01 -->
  <!-- Author: David Marston -->
  <!-- Purpose: Test of numbering of multi-level document with no attributes specified. -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="DocFrag" place="id(number)" -->
  <!-- Scenario: operation="standard-XML" -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="title">
  <xsl:number/><xsl:text>. </xsl:text><xsl:value-of select="."/><xsl:text>
</xsl:text>
</xsl:template>
 
<xsl:template match="text()"><!-- To suppress empty lines --></xsl:template>
 
</xsl:stylesheet>
