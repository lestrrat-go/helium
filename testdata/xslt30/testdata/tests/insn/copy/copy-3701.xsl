<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test copy-of identity transformation on XML with namespace nodes
       that redefines the default and one prefixed namespace on inner element. -->

  <!-- The data also has a namespace node that is re-declared exactly the same on the inner element.
    This redundant declaration does not have to be copied, so it isn't. -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="/">
    <xsl:copy-of select="."/>
</xsl:template>

</xsl:stylesheet>