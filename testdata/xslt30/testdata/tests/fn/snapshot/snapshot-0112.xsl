<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" expand-text="1">

<xsl:strip-space elements="*"/>

<xsl:template match="/">
  <xsl:variable name="s" as="node()" select="snapshot(//*:boundedBy)/root()"/>
  <out>
    <a>{count(distinct-values($s//node()/(.|@*|namespace::*)/generate-id())) = count($s//node()/(.|@*|namespace::*))}</a>
    <a>{not($s//node()/(.|@*|namespace::*)/generate-id() = //node()/(.|@*|namespace::*)/generate-id())}</a>
  </out>
</xsl:template>



</xsl:stylesheet>