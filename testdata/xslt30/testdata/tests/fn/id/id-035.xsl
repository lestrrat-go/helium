<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="document/departaments">
      <body>
        <xsl:for-each select="departament">
          <dep name="{@nam}">
          <ul>
            <xsl:for-each select="id(@people)">
              <li>
                <xsl:value-of select="@name"/>
              </li>
            </xsl:for-each>
          </ul>
          </dep>
        </xsl:for-each>
      </body>
  </xsl:template>
</xsl:stylesheet>
