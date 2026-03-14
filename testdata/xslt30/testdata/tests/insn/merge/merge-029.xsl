<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <xsl:mode on-no-match="shallow-copy"/>
  
  <xsl:strip-space elements="*"/>
  
  <xsl:variable name="original" select="doc('merge018-current.xml')"/>
  <xsl:variable name="updates" select="doc('merge018-updates.xml')"/>
  
  <xsl:template match="/" name="main">
    <xsl:apply-templates select="$original/*"/>
  </xsl:template>  
  
  <xsl:template match="Addresses">
    <Addresses>
      <xsl:merge>
        <xsl:merge-source for-each-item="$updates, $original"
                        select=".//Address">
          <xsl:merge-key select="AddressesId"/>
        </xsl:merge-source>
        <xsl:merge-action>
          <xsl:variable name="status" select="
            if (count(current-merge-group()) = 1)
            then if (current-merge-group()[1]/root(.) is $original) then 'Deleted' else 'New'
            else if (deep-equal(current-merge-group()[1], current-merge-group()[2])) then 'NoChange' else 'Updated'"/>
          <Address status="{$status}">
            <xsl:copy-of select="current-merge-group()[1]/(AddressesId, Street)"/>
          </Address>
        </xsl:merge-action>
      </xsl:merge>
    </Addresses>
  </xsl:template>
</xsl:stylesheet>  
              