<?spec xslt#grouping?>
<!-- Use case for multilevel grouping to do a hierarchic inversion -->

<not-html xsl:version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <body>
    <h2>
      <xsl:text>Bug Summary (</xsl:text>
      <xsl:value-of select="count(bugs/bug)"/>
      <xsl:text>)</xsl:text>
    </h2>
    <table>
      <tr>
        <xsl:for-each-group select="bugs/bug" group-by="@team">
          <xsl:sort select="count(current-group())"
                    data-type="number" order="descending"/>
          <th>
            <xsl:value-of select="@team"/>
            <xsl:text> (</xsl:text>
            <xsl:value-of select="count(current-group())"/>
            <xsl:text>)</xsl:text>
          </th>
        </xsl:for-each-group>
      </tr>
      <tr>
        <xsl:for-each-group select="bugs/bug" group-by="@team">
          <xsl:sort select="count(current-group())"
                    data-type="number" order="descending"/>
          <td>
            <xsl:for-each-group select="current-group()" group-by="@dev">
              <xsl:sort select="count(current-group())"
                        data-type="number" order="descending"/>
              <xsl:value-of select="@dev"/>
              <xsl:text> (</xsl:text>
              <xsl:value-of select="count(current-group())"/>
              <xsl:text>)</xsl:text>
              <xsl:if test="position() != last()"><br/></xsl:if>
            </xsl:for-each-group>
          </td>
        </xsl:for-each-group>
      </tr>
    </table>
  </body>
</not-html>