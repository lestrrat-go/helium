<?xml version="1.0"?>

<!-- test use of xsl:element with a namespace prefix not known until run-time -->
<?spec xslt#xsl-element?>

<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:a="http://a/" xmlns:b="http://b/"
  xmlns:c="http://c/">

  <xsl:template match="/">
    <out>
      <without-default-namespace>
        <xsl:for-each select="//*">
          <xsl:variable name="name" select="name()"/>
          <xsl:element name="{$name}">xxx</xsl:element>
        </xsl:for-each>
      </without-default-namespace>
      <with-default-namespace xmlns="http://default/">
        <xsl:for-each select="//*">
          <xsl:variable name="name" select="name()"/>
          <xsl:element name="{$name}">xxx</xsl:element>
        </xsl:for-each>
      </with-default-namespace>
    </out>
  </xsl:template>

</xsl:stylesheet>
