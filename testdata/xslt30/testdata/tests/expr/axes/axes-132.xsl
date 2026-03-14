<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:f="http://f/" exclude-result-prefixes="f" xpath-default-namespace="http://banana/">

<?spec fo#func-node-name?>
<!-- Purpose: Test node-name function on all kinds of node (and none). -->
<?same-as-1.0 no?>

<xsl:output method="xml" encoding="UTF-8" indent="no" />

<xsl:template match="/">
  <out>
    <a/><xsl:value-of select="f:nn(/a)"/>;
    <a/><xsl:value-of select="f:nn(//f/@custard)"/>;
    <a/><xsl:value-of select="f:nn(//f/@xml:space)"/>;
    <a/><xsl:value-of select="f:nn(/processing-instruction()[1])"/>;
    <a/><xsl:value-of select="f:nn(/comment()[1])"/>;
    <a/><xsl:value-of select="f:nn(/a/namespace::xml[1])"/>;
    <a/><xsl:value-of select="f:nn(//f/text())"/>;
    <a/><xsl:value-of select="f:nn(())"/>;                           
  </out>
</xsl:template>

<xsl:function name="f:nn">
  <xsl:param name="n" as="node()?"/>
  <xsl:variable name="z" select="node-name($n)"/>
  <xsl:sequence select="local-name($n), '|', count($n), count($z), '|', 
                        namespace-uri-from-QName($z), local-name-from-QName($z)"/>
</xsl:function>

</xsl:stylesheet>