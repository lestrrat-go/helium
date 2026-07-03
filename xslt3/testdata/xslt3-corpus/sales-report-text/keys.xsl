<?xml version="1.0" encoding="UTF-8"?>
<!-- Included module declaring the xsl:key used by the main module. -->
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <xsl:key name="region-by-code" match="region" use="@code"/>

</xsl:stylesheet>
