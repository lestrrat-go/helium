<?spec xslt#grouping?>
<xsl:transform version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <table>
      <!-- Reference current-group() within group-by expression -->
      <xsl:for-each-group select="cities/city" group-by="xs:decimal(@pop)">
        <pop value="{current-grouping-key()}">
          <xsl:for-each-group select="current-group()" group-by="@name[../@pop = current-group()/@pop]">
            <city name="{@name}"/>
          </xsl:for-each-group>
        </pop>
      </xsl:for-each-group>
    </table>
  </xsl:template>
</xsl:transform>
