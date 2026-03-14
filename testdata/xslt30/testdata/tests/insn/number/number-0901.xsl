<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- CaseName: collation01 -->
  <!-- Author: David Marston -->
  <!-- Purpose: Test of greek-numeral "traditional" sequence. -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[2]/item[3]/p[1]/text()[6]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(number)/ulist[1]/item[3]/p[1]/text()[1]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(convert)/ulist[1]/item[6]/p[1]/text()[1]" -->
  <!-- SpecCitation: Rec="XSLT" Version="1.0" type="OASISptr1" place="id(convert)/p[5]/text()[6]" -->
  <!-- Scenario: operation="standard-HTML" -->
  <!-- Discretionary: number-greek-trad="true" -->

<xsl:output method="xml" encoding="ISO-8859-1" indent="no"/>

<xsl:template match="doc">
  <!-- Saxon does not implement "traditional Greek" numbering
       and from the discussion around http://lists.w3.org/Archives/Public/www-style/2002Dec/0072.html,
       this seems like a wise decision -->
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="note">
  <xsl:number level="any" from="chapter" format="&#x03b1;" letter-value="traditional"/>
  <xsl:apply-templates/>
</xsl:template>

</xsl:stylesheet>
