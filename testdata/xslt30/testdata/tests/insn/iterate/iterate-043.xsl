<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="#all"
  expand-text="true"
  version="3.0">
  
  <xsl:template name="xsl:initial-template">
        <section>
            <xsl:iterate select="1 to 5">
              <xsl:param name="p1" as="xs:integer" select="4"/>
              <xsl:on-completion select="$p1"/>
              <xsl:variable name="p1" select="$p1+2"/>
              <xsl:next-iteration>
                <xsl:with-param name="p1" select="$p1+3"/>
              </xsl:next-iteration>
            </xsl:iterate>
        </section>      
  </xsl:template>
</xsl:stylesheet>