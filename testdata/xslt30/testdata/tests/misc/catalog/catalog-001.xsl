<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:cat="http://www.w3.org/2012/10/xslt-test-catalog" exclude-result-prefixes="cat">
  
  <!-- Test that the test catalog is valid with respect to its schema -->


  <xsl:import-schema namespace="http://www.w3.org/2012/10/xslt-test-catalog"/>

  <xsl:template match="/">
    <xsl:choose>
      <xsl:when test="* instance of schema-element(cat:catalog)">
        <ok>
          <xsl:for-each select="document(*/cat:test-set/@file)">
            <xsl:try>
              <xsl:variable name="copy">
                <xsl:copy-of select="." validation="strict"/>
              </xsl:variable>
              <xsl:sequence select="$copy[2]"/>
              <!-- empty sequence -->
              <xsl:catch>
                <invalid doc="{document-uri(.)}"/>
              </xsl:catch>
            </xsl:try>
          </xsl:for-each>
        </ok>
      </xsl:when>
      <xsl:otherwise>
        <not-ok/>
      </xsl:otherwise>
    </xsl:choose>
  </xsl:template>

</xsl:stylesheet>
