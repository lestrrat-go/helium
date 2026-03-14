<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xmlns:z="http://schema076.uri/"
        exclude-result-prefixes="xsi">

<!-- PURPOSE: Test that string-length(.) fails if typed value isn't a string -->
<?spec fo#func-string-length?>

<xsl:import-schema namespace="http://schema076.uri/" schema-location="schema076.xsd"/>



  <xsl:template match="/">
    <xsl:variable name="x">
        <xsl:document validation="preserve">
            <n xsl:type="z:numbers">1.000 2.000 3.000 4.000</n>
        </xsl:document>
    </xsl:variable>
    <out>
      <xsl:for-each select="$x/n">
          <l><xsl:value-of select="string-length()"/></l>
          <l><xsl:value-of select="string-length(.)"/></l>
      </xsl:for-each>
    </out>
  </xsl:template>
  
</xsl:stylesheet>