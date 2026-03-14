<?xml version="1.0" encoding="UTF-8"?>

<!-- call an overloaded extension function within xsl:evaluate -->
<!-- bug 2859861 -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:Math="java:java.lang.Math" version="3.0">
  
  <xsl:output method="xml" encoding="UTF-8"/>
  
  <xsl:template match="/" name="main">
    <Result>
      <xsl:evaluate xpath="'Math:abs(-1)'" use-when="function-available('Math:abs', 1)"/>
      <xsl:evaluate xpath="'abs(-1)'" use-when="not(function-available('Math:abs', 1))"/>
    </Result>
  </xsl:template>
</xsl:stylesheet> 
