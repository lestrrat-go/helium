<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#for-each?>
  <!-- Section: 8 Repetition -->
  <!-- Purpose: Test for-each example from XSLT spec. -->

<xsl:template match="/">
  <html>
    <head>
      <title>Customers</title>
    </head>
    <body>
      <table>
	<tbody>
	  <xsl:for-each select="customers/customer">
	    <tr>
	      <th>
		<xsl:apply-templates select="name"/>
	      </th>
	      <xsl:for-each select="order">
		<td>
		  <xsl:apply-templates/>
		</td>
	      </xsl:for-each>
	    </tr>
	  </xsl:for-each>
	</tbody>
      </table>
    </body>
  </html>
</xsl:template>

</xsl:stylesheet>
