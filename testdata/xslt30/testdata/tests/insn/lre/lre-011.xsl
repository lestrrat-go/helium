<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#literal-text-nodes?>
    <!-- Purpose: Try to put out text without any tags.-->

<xsl:output method="text"/>

<xsl:template match="doc">
  This should be directly at the top.
</xsl:template>

</xsl:stylesheet>
