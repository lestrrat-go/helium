<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  exclude-result-prefixes="#all"
  expand-text="yes"
  version="3.0">
  
  <xsl:mode streamable="yes" on-no-match="shallow-copy" use-accumulators="#all"/>
  
  <xsl:strip-space elements="*"/>
  <xsl:output indent="yes"/>
  
  <xsl:accumulator name="inventtransid" as="xs:string?" initial-value="()" streamable="yes">
    <xsl:accumulator-rule match="CustInvoiceTrans" select="()"/>
    <xsl:accumulator-rule match="CustInvoiceTrans/InventTransId/text()" select="string()"/>
  </xsl:accumulator>
  
  <xsl:accumulator name="map-inventtrans-to-itemid" as="map(xs:string, xs:integer)" initial-value="map{}" streamable="yes">
    <xsl:accumulator-rule match="CustInvoiceTrans/ItemId/text()" select="map:put($value, accumulator-before('inventtransid'), xs:integer(.))"/>
  </xsl:accumulator>
  
  <xsl:template match="/*">
    <xsl:copy>
      <xsl:fork>
        <xsl:sequence>
          <xsl:for-each-group select="CustInvoiceTrans!copy-of()" group-by="ItemId">
            <xsl:copy>
              <xsl:apply-templates select="@*"/>
              <xsl:apply-templates>
                <xsl:with-param name="sum" tunnel="yes" select="sum(current-group()/LineAmount)"/>
              </xsl:apply-templates>
            </xsl:copy>
          </xsl:for-each-group>
        </xsl:sequence>
        <xsl:sequence>
          <xsl:for-each-group select="TaxTrans!copy-of()" group-by="accumulator-before('map-inventtrans-to-itemid')(InventTransId)">
            <xsl:copy>
              <xsl:apply-templates select="@*"/>
              <xsl:apply-templates>
                <xsl:with-param name="sums" tunnel="yes" select="map { 'TaxAmount' : sum(current-group()/TaxAmount), 'TaxBaseAmount' : sum(current-group()/TaxBaseAmount) }"/>
              </xsl:apply-templates>
            </xsl:copy>
          </xsl:for-each-group>
        </xsl:sequence>
      </xsl:fork>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="CustInvoiceTrans/LineAmount">
    <xsl:param name="sum" tunnel="yes"/>
    <xsl:copy>{$sum}</xsl:copy>
  </xsl:template>
  
  <xsl:template match="TaxTrans/TaxAmount">
    <xsl:param name="sums" tunnel="yes"/>
    <xsl:copy>{$sums?TaxAmount}</xsl:copy>
  </xsl:template>
  
  <xsl:template match="TaxTrans/TaxBaseAmount">
    <xsl:param name="sums" tunnel="yes"/>
    <xsl:copy>{$sums?TaxBaseAmount}</xsl:copy>
  </xsl:template>
  
</xsl:stylesheet>