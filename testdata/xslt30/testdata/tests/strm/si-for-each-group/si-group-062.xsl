<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math"
  version="3.0">

  <!-- Non-streamable xsl:source-document instruction in a streamable template -->
  
  <xsl:mode streamable="yes"/>

  <xsl:output indent="no"/>
  <xsl:strip-space elements="*"/>

  <xsl:template match="Orders">
    <xsl:copy>
      <xsl:for-each-group select="Order/copy-of()" group-adjacent="@number">
        <group number="{current-grouping-key()}">
          <xsl:source-document href="../docs/transactions.xml" streamable="no">
            <xsl:for-each select="//transaction[@date = current-group()[1]/Date]">
              <value account="{../account-number}">
                <xsl:value-of select="@value"/>
              </value>
            </xsl:for-each>
          </xsl:source-document>
        </group>
      </xsl:for-each-group>
    </xsl:copy>
  </xsl:template>

</xsl:stylesheet>
