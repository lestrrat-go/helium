<xsl:transform version="3.0"
  xmlns:example="https://example.com/ns"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <xsl:template match="/">
    <example:source-document href="." streamable="{if (true()) then 'yes' else 'no'}">
      <foo>
        <xsl:call-template name="some-template"/>
      </foo>
    </example:source-document>
  </xsl:template>
  
  <xsl:template name="some-template"/>
</xsl:transform>