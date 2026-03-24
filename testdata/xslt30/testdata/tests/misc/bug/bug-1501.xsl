<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- test copying an attribute in non-default namespace (5.3.2/004) -->
  <!-- test generates two attributes with same prefix but different URI; one prefix must change -->


  <xsl:template match="in">
    <out>
      <xsl:variable name="atts" as="element()">
        <e>
          <xsl:copy-of select="foo/@* | bar/@*"/>
        </e>
      </xsl:variable>
      <ok>
        <xsl:value-of select="count(distinct-values($atts//@*/name())) = 2"/>
      </ok>
    </out>
  </xsl:template>

</xsl:stylesheet>
