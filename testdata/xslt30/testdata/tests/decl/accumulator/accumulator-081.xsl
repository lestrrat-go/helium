<xsl:transform version="3.0" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <xsl:variable name="data">
    <element/>
  </xsl:variable>

  <xsl:mode on-no-match="shallow-skip" use-accumulators="#all"/>

  <xsl:accumulator name="accumulator" as="xs:integer" initial-value="0">
    <xsl:accumulator-rule match="element">
      <xsl:sequence select="1"/>
    </xsl:accumulator-rule>
    <xsl:accumulator-rule match="element">
      <xsl:sequence select="2"/>
    </xsl:accumulator-rule>
  </xsl:accumulator>

  <xsl:template name="xsl:initial-template">
    <out>
      <xsl:for-each select="$data">
        <xsl:apply-templates/>
        <xsl:sequence select="accumulator-after('accumulator')"/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:transform>
