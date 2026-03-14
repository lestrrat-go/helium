<?spec xslt#grouping?>
<!-- test value-based grouping, under conditions of non-transitivity: see erratum E25 -->
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">
    
    <xsl:template name="main" match="/">
      <out>
        <xsl:variable name="groups" as="element(group)*">
           <xsl:for-each-group group-by="." select="
                 xs:float('1.0'),
                 xs:decimal('1.0000000000100000000001'),
                 xs:double( '1.00000000001')">
              <group><xsl:value-of select="current-group()"/></group>
          </xsl:for-each-group>
        </xsl:variable>
        <one><xsl:value-of select="count($groups)"/></one>
        <xsl:variable name="groups" as="element(group)*">
           <xsl:for-each-group group-by="." select="
                 xs:double( '1.00000000001'),
                 xs:float('1.0'),
                 xs:decimal('1.0000000000100000000001'),
                 xs:double( '1.00000000001')">
              <group><xsl:value-of select="current-group()"/></group>
          </xsl:for-each-group>
        </xsl:variable>
        <one><xsl:value-of select="count($groups)"/></one>
        <xsl:variable name="groups" as="element(group)*">
           <xsl:for-each-group group-by="." select="
                 xs:decimal('1.0000000000100000000001'),
                 xs:float('1.0'),
                 xs:decimal('1.0000000000100000000001'),
                 xs:double( '1.00000000001')">
              <group><xsl:value-of select="current-group()"/></group>
          </xsl:for-each-group>
        </xsl:variable>
        <one><xsl:value-of select="count($groups)"/></one>
      </out>
     </xsl:template>        
</xsl:stylesheet>