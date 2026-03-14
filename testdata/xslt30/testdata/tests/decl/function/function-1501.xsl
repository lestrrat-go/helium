<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
 xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">
 <xsl:output method="xml" encoding="UTF-8" indent="no" omit-xml-declaration="yes"/>
 <xsl:template match="doc">
  <!-- Testing fn:available-environment-variables#0 and fn:environment-variable#1-->
  <?spec fo#func-available-environment-variables?>
  <?spec fo#func-get-available-environment-variable?>
  <x>
   <xsl:value-of select="
    every $s in ('user', 'USER', 'shell', 'SHELL', 'path', 'PATH', 'zzz', '', '%$£*!') satisfies
          if (available-environment-variables() = $s) 
          then exists(environment-variable($s)) 
          else empty(environment-variable($s))"/>
  </x>
 </xsl:template>
</xsl:stylesheet>
