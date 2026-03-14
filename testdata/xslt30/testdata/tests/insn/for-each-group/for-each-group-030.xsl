<?spec xslt#grouping?>
<xsl:transform version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <!-- test group-ending-with to merge continuation records -->

  <xsl:template match="body">
    <chapter>
      <xsl:for-each-group select="*" group-ending-with="*[@cont='no']">
        <section>
          <xsl:for-each select="current-group()">
            <para>
              <xsl:value-of select="."/>
            </para>
          </xsl:for-each>
        </section>
      </xsl:for-each-group>
    </chapter>
  </xsl:template>

</xsl:transform>
