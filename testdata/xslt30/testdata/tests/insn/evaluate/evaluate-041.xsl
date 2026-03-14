<?xml version="1.0" encoding="UTF-8"?>

<!-- Contents of @with-params change on subsequent calls, but the extra params are not used-->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"  version="3.0"
exclude-result-prefixes="xs">

  <xsl:param name="exp">$p1 + 1</xsl:param>
  
  <xsl:template match="/" name="main">
    <Result>
      <xsl:for-each select="2 to 10">
        <xsl:variable name="r" as="xs:integer">
          <xsl:evaluate xpath="$exp"
                        with-params="map{xs:QName('p'||.): position() mod 5}">
            <xsl:with-param name="p1" select="20"/>
          </xsl:evaluate>
        </xsl:variable>
        <out><xsl:value-of select="$r"/></out>
      </xsl:for-each>
    </Result>
  </xsl:template>
  
</xsl:stylesheet> 
