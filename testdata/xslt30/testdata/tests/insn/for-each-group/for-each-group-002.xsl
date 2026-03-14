<?spec xslt#grouping?>
<table xsl:version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<!-- simple test of value-based grouping -->
  <tr>
    <th>Position</th>
    <th>Country</th>
    <th>City List</th>
    <th>Population</th>
  </tr>
  <xsl:for-each-group select="cities/city" group-by="@country" bind-group="g">
    <tr>
      <td><xsl:value-of select="position()"/></td>
      <td><xsl:value-of select="@country"/></td>
      <td>
        <xsl:for-each select="$g">
          <xsl:value-of select="@name"/>
          <xsl:if test="position() != last()">, </xsl:if>
        </xsl:for-each>
      </td>
      <td><xsl:value-of select="sum($g/@pop)"/></td>
    </tr>
  </xsl:for-each-group>
</table>
