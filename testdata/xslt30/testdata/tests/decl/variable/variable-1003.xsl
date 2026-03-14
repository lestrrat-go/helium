<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Import precedence applies to top-level params
     (even when template referencing it is in the imported file) -->

  <xsl:import href="variable-1003a.xsl"/>

  <xsl:param name="test" select="'main stylesheet, should have highest precedence'"/>

</xsl:stylesheet>
