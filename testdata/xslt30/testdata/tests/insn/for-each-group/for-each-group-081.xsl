<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet exclude-result-prefixes="xs" version="2.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <!-- Effect of current-group() when absent: differs between 2.0 and 3.0 processors -->
  
  <xsl:output encoding="UTF-8" indent="yes" method="xml"/>
  
  <xsl:variable name="in">
    <root>
      <line>
        <value>50</value>
      </line>
    </root>
  </xsl:variable>
  
  <xsl:template name="main">
    <xsl:apply-templates select="$in"/>
  </xsl:template>
  
  <xsl:template match="root">
    <out>
      <xsl:for-each select="line">
        <xsl:call-template name="line"/>
      </xsl:for-each>
    </out>
  </xsl:template>
  
  <xsl:template name="line">
    <test>
      <xsl:value-of select="empty(current-group())"/>
    </test>
  </xsl:template>
</xsl:stylesheet>