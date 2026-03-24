<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#import?>
<?spec xslt#include?>
    <!-- Purpose: Included document's xsl:import (i23incl imports i23sub) is moved into the
    including document. Import precedence for templates matching foo is
    (high) i23sub, h (low) -->

<xsl:import href="h.xsl"/>
<xsl:include href="fragments/i23incl.xsl"/><!-- last one on list has higher precedence -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="foo"/>
  </out>
</xsl:template>

</xsl:stylesheet>