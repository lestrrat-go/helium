<?xml version="1.0" encoding="UTF-8"?>
<!-- A reusable XSLT 3.0 package served to xsl:use-package by name through the
     confined PackageResolver. It exposes one public function. -->
<xsl:package name="urn:helium:corpus:text-utils" package-version="1.0" version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:tu="urn:helium:corpus:text-utils-fn"
    exclude-result-prefixes="xs tu">

  <!-- Uppercased first letter of a string, used as a grouping key. -->
  <xsl:function name="tu:initial" as="xs:string" visibility="public">
    <xsl:param name="s" as="xs:string"/>
    <xsl:sequence select="upper-case(substring(normalize-space($s), 1, 1))"/>
  </xsl:function>

</xsl:package>
