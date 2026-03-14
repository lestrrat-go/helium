<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- validate element with type="xs:untypedAtomic" unsuccessfully -->

  <xsl:template name="main">
      <z xsl:type="xs:untypedAtomic">abcd<a/>wxyz</z>
  </xsl:template>

</xsl:stylesheet>
