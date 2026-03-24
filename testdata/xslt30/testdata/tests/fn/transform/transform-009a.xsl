<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs"
  expand-text="yes">

  <xsl:output method="text"/>

  <xsl:template match="/">

    <xsl:result-document method="xml">
      <principal>main output</principal>
    </xsl:result-document>

    <xsl:result-document href="secondary1.txt" method="text">
      <secondary1>text doc</secondary1>
    </xsl:result-document>

    <xsl:result-document href="secondary2.xml" method="xml" omit-xml-declaration="yes">
      <secondary2>xml doc</secondary2>
    </xsl:result-document>

  </xsl:template>

</xsl:stylesheet>

