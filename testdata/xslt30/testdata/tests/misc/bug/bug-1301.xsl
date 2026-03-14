<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec xslt#sorting?>
<!-- BUG: not a bug, as it turned out. Tests numeric sorting. -->

    <xsl:template match="PERIODIC_TABLE">
      <html>
        <head>
          <title>Atomic Number vs. Atomic Weight</title>
        </head>
        <body>
          <h1>Atomic Number vs. Atomic Weight</h1>
          <table>
            <th>Element</th>
            <th>Atomic Number</th>
            <th>Atomic Weight</th>
            <xsl:apply-templates>
              <xsl:sort select="ATOMIC_NUMBER" data-type="number" order="ascending"/>
            </xsl:apply-templates>
          </table>
        </body>
      </html>
    </xsl:template>
    
    <xsl:template match="ATOM">
      <tr>
        <td><xsl:apply-templates select="NAME"/></td>
        <td><xsl:apply-templates select="ATOMIC_NUMBER"/></td>
        <td><xsl:apply-templates select="ATOMIC_WEIGHT"/></td>
      </tr>
    </xsl:template>
    
</xsl:stylesheet>

