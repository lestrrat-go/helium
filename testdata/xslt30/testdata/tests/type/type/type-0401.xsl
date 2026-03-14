<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <!-- PURPOSE: Test "instance of" on dynamic types. -->

  <xsl:template match="/">
    <out>
      <xsl:variable name="c" select="(doc = 1) cast as xs:boolean"/>
      <t><xsl:value-of select="(if ($c) then 1.0 else 'x') instance of xs:decimal"/></t>
      <t><xsl:value-of select="(if ($c) then true() else 'x') instance of xs:boolean"/></t>
      <t><xsl:value-of select="(if ($c) then 'Paris' else 13) instance of item()"/></t>
      <t><xsl:value-of select="(for $i in 1 to xs:integer((doc cast as xs:decimal)+10) return 3) instance of xs:decimal +"/></t>
    </out>
  </xsl:template>

</xsl:stylesheet>