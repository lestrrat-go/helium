<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:axsl="http://www.w3.org/1999/XSL/TransformAlias"
  version="2.0">

<xsl:namespace-alias result-prefix="xsl" stylesheet-prefix="axsl" />

    <!-- Purpose: Get xmlns declaration attached to outermost LRE.
     Prefix "axsl" should also be literal. -->

<xsl:template match="/">
  <axsl:template match="/"/>
</xsl:template>

</xsl:stylesheet>
