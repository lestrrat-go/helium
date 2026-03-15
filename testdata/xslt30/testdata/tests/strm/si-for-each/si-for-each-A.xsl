<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
   
  <!-- within xsl:source-document, use xsl:for-each -->
   
  <xsl:template name="fe-001" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each select="account/transaction[@value &lt; 0]">
          <xsl:copy-of select="."/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:for-each with numbering -->
   
  <xsl:template name="fe-002">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each select="account/transaction[@value &lt; 0]">
          <transaction nr="{position()}">
            <xsl:copy-of select="@*"/>
          </transaction>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:for-each with positional selection -->
   
  <xsl:template name="fe-003">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each select="account/transaction[position() &lt; 5]">
          <transaction nr="{position()}">
            <xsl:copy-of select="@*"/>
          </transaction>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- xsl:for-each over atomized nodes -->
   
  <xsl:template name="fe-004">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each select="subsequence(data(account/transaction/@value), 5, 3)">
          <transaction nr="{position()}">
            <xsl:value-of select="."/>
          </transaction>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- crawling selection, inspection body -->
   
  <xsl:template name="fe-005" expand-text="yes">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="(//*)[position()=1 to 6]">{position()}:{name()} </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- crawling selection, inspection body -->
   
  <xsl:template name="fe-006">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="//*">
          <xsl:sequence select="count(ancestor-or-self::*)"/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- xsl:source-document / xsl:for-each / xsl:copy-of with a predicate -->
  
  <xsl:template name="fe-007">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="head(/BOOKLIST/BOOKS/ITEM)">
            <xsl:copy-of select="*[namespace-uri(.) = '']"/>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
  </xsl:template>
  
  <!-- within xsl:source-document, use parent axis within xsl:for-each -->
   
  <xsl:template name="fe-008">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each select="account/transaction[position() lt 5]">
          <xsl:sequence select="name(..)"/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use ancestor axis within xsl:for-each -->
   
  <xsl:template name="fe-009">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each select="subsequence(account/transaction, 1, 4)">
          <xsl:sequence select="name(ancestor::*[1])"/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use subsequence() within xsl:for-each/@select -->
   
  <xsl:template name="fe-010">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each select="subsequence(account/transaction, 1, 4)">
          <xsl:copy-of select="."/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, select attributes within xsl:for-each/@select -->
   
  <xsl:template name="fe-011">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each select="account/transaction/@value">
          <xsl:if test=". >= 0">
            <xsl:sequence select="string(.)"/>
          </xsl:if>  
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, iterate over ancestor axis within xsl:for-each/@select -->
   
  <xsl:template name="fe-012">
    <out xmlns="http://loan.shark.com/">
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:for-each select="outermost(//*:extra)">
          <xsl:for-each select="ancestor::*">
            <xsl:copy>
              <xsl:copy-of select="@*"/>
            </xsl:copy>
          </xsl:for-each>  
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- xsl:source-document / xsl:for-each / xsl:copy-of with a predicate and some local variables-->
  
  <xsl:template name="fe-013">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="head(/BOOKLIST/BOOKS/ITEM)">
            <xsl:variable name="x" select="current-date()"/>
            <xsl:copy-of select="let $y := current-dateTime() return *[namespace-uri(.) != string($x) and local-name(.) != string($y) and name(.) != concat($x, $y)]"/>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
  </xsl:template>       

       
</xsl:transform>

