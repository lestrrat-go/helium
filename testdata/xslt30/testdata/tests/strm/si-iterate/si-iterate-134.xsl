<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  exclude-result-prefixes="xs map" version="3.0">
  <xsl:template name="main">
    <xsl:source-document streamable="yes" href="../docs/citygml.xml">
      <elements>
        <xsl:variable name="histogram" as="map(xs:string, xs:integer)">
          <xsl:iterate select=".//*/name()">
            <xsl:param name="m" as="map(xs:string, xs:integer)" select="map{}"/>
            <xsl:on-completion>
              <xsl:sequence select="$m"/>
            </xsl:on-completion>
            <xsl:variable name="count" as="xs:integer?" select="($m(.), 0)[1]"/>
            <xsl:next-iteration>
              <xsl:with-param name="m" select="map:merge((map:entry(., $count+1), $m))"/>
            </xsl:next-iteration>           
          </xsl:iterate>
        </xsl:variable>
        <xsl:for-each select="map:keys($histogram)">
          <e name="{.}" count="{$histogram(.)}"/>
        </xsl:for-each>
      </elements>
    </xsl:source-document>
  </xsl:template>
</xsl:stylesheet>
