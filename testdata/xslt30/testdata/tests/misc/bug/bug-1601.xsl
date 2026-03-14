<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <?spec xslt#copying?>
  <!-- BUG: Saxon 5.3.2/004 -->
  <!-- DESCRIPTION: If an attribute node is copied using xsl:copy or xsl:copy-of, and it is in a
    non-default namespace, the result file may contain a reference to a namespace that
    is not declared. -->


  <xsl:template match="in">
    <out>
      <xsl:variable name="atts" as="element()">
        <e>
          <xsl:for-each select="foo/@* | bar/@*">
            <xsl:copy/>
          </xsl:for-each>
        </e>
      </xsl:variable>
      <ok>
        <xsl:value-of select="count(distinct-values($atts//@*/name())) = 2"/>
      </ok>
    </out>
  </xsl:template>

</xsl:stylesheet>
