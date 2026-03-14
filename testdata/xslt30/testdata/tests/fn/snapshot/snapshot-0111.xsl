<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" expand-text="1">

<xsl:strip-space elements="*"/>

<xsl:template match="/">
  <xsl:variable name="s" as="node()" select="snapshot(//*:boundedBy)/.."/>
  <out>
    <a actual="{count($s/ancestor::node())}">{count($s/ancestor::node()) = 1}</a>
    <a actual="{count($s/preceding-sibling::node())}">{count($s/preceding-sibling::node()) = 0}</a>
    <a actual="{count($s/following-sibling::node())}">{count($s/following-sibling::node()) = 0}</a>
    <a actual="{count($s/preceding::node())}">{count($s/preceding::node()) = 0}</a>
    <a actual="{count($s/following::node())}">{count($s/following::node()) = 0}</a>
    <a actual="{count($s/parent::node())}">{count($s/parent::node()) = 1}</a>
    <a actual="{count($s/child::node())}">{count($s/child::node()) = 1}</a>
    <a actual="{count($s/descendant::*)}">{count($s/descendant::*) = 4}</a>
    <a actual="{count($s/descendant::node())}">{count($s/descendant::node()) = 6}</a>
    <a actual="{count($s/attribute::*)}">{count($s/attribute::*) = 1}</a>
    <a actual="{count($s/namespace::*)}">{count($s/namespace::*) = 3}</a>
  </out>
</xsl:template>



</xsl:stylesheet>