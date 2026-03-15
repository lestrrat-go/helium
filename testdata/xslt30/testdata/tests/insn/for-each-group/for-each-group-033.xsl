<?spec xslt#grouping?>
<xsl:transform version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- Test group-by where an item goes in more than one group -->

  <xsl:template match="/">
    <g>
      <xsl:for-each-group select="//city" group-by="number(@pop), string-length(@name)">
        <group>
          <xsl:copy-of select="current-group()"/>
        </group>
      </xsl:for-each-group>
    </g>
  </xsl:template>

</xsl:transform>
