<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"  xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0">

   <!-- Static param with invalid static attribute -->
   
   <xsl:param name="static-param" static="YES" select="'public'" />

   <xsl:template name="xsl-initial-template" />
   
</xsl:transform>
