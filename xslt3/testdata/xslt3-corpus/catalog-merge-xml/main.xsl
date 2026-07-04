<?xml version="1.0" encoding="UTF-8"?>
<!-- Entry module for the catalog-merge-xml case.
     Exercises: multi-document input via document() (a second file fetched
     through the confined runtime URIResolver), apply-templates modes,
     params, and indented XML output. -->
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs">

  <xsl:output method="xml" encoding="UTF-8" indent="yes"/>

  <xsl:param name="storeName" as="xs:string" select="'Helium Store'"/>
  <xsl:param name="taxRate" as="xs:double" select="0.0"/>

  <!-- Second input document, merged in at runtime. -->
  <xsl:variable name="products" select="document('products.xml')"/>

  <xsl:template match="/orders">
    <invoices store="{$storeName}">
      <xsl:apply-templates select="order"/>
    </invoices>
  </xsl:template>

  <xsl:template match="order">
    <invoice id="{@id}" customer="{@customer}">
      <xsl:apply-templates select="line" mode="detail"/>
      <xsl:variable name="subtotal" select="sum(for $l in line return
          $l/@qty * $products/products/product[@id = $l/@product]/@price)"/>
      <total subtotal="{format-number($subtotal, '0.00')}"
             tax="{format-number($subtotal * $taxRate, '0.00')}"
             grand="{format-number($subtotal * (1 + $taxRate), '0.00')}"/>
    </invoice>
  </xsl:template>

  <xsl:template match="line" mode="detail">
    <xsl:variable name="p" select="$products/products/product[@id = current()/@product]"/>
    <item product="{@product}" name="{$p/@name}" qty="{@qty}"
          unitPrice="{$p/@price}"
          lineTotal="{format-number(@qty * $p/@price, '0.00')}"/>
  </xsl:template>

</xsl:stylesheet>
