<?spec xslt#grouping?>
<!-- Grouping used to arrange data in columns in a table -->
<!-- Doesn't yet handle the &nbsp for missing cell -->
<table xsl:version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:variable name="columns" select="2"/>
  <xsl:variable name="items" select="count(cities/city)"/>
  <xsl:variable name="items-per-column" 
                select="ceiling($items div $columns)"/>
 <xsl:variable name="sorted-cities">
     <xsl:for-each select="cities/city">
     <xsl:sort select="@name"/>
        <xsl:copy-of select="."/>
     </xsl:for-each>
 </xsl:variable>
  <xsl:for-each-group select="$sorted-cities/city" group-by="position() mod $items-per-column">
    <tr>
      <xsl:for-each select="current-group()">
        <td><xsl:value-of select="@name"/></td>
      </xsl:for-each>
    </tr>
  </xsl:for-each-group>
</table>    