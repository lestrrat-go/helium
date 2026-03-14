<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- CaseName: numbering75 -->
  <!-- Author: David Marston -->
  <!-- Purpose: Test xsl:number with level=multiple, from specifies two levels. -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[2]/item[2]/p[1]/text()[5]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[1]/item[2]/p[1]/text()[3]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[1]/item[3]/p[1]/text()[1]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(convert)/ulist[1]/item[1]/p[1]/text()[1]" -->
  <!-- Scenario: operation="standard-XML" -->

<xsl:template match="doc">
  <out><xsl:apply-templates/></out>
</xsl:template>

<xsl:template match="note">
  <xsl:number level="multiple" from="doc|chapter" format="1"/><xsl:text>: </xsl:text><xsl:value-of select="."/><xsl:text>
</xsl:text>
</xsl:template>

<xsl:template match="text()"><!-- To suppress empty lines --></xsl:template>

</xsl:stylesheet>
