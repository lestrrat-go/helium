<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs err">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/> 
   
  <!-- within xsl:stream, use multiply-consuming map expression -->
  
  <xsl:template name="m-001" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="BOOKLIST/BOOKS/ITEM">
          <xsl:variable name="m" as="map(*)"
            select="map{ 'author': string(AUTHOR),
                         'title': string(TITLE),
                         'price': number(PRICE),
                         'publisher': string(PUBLISHER)}"/>
          <book author="{$m('author')}" title="{$m('title')}" price="{$m('price')}" publisher="{$m('publisher')}"/>     
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:stream, use map expression with multi-valued map entries -->
  
  <xsl:template name="m-002" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
          <xsl:variable name="m" as="map(*)" select="map{
          		'authors':/BOOKLIST/BOOKS/ITEM/AUTHOR/string(),
          		'titles':/BOOKLIST/BOOKS/ITEM/TITLE/string(),
          		'prices':/BOOKLIST/BOOKS/ITEM/PRICE/number(),
          		'publishers':/BOOKLIST/BOOKS/ITEM/PUBLISHER/string()}"/>
          <books authors="{$m('authors')}" titles="{$m('titles')}" prices="{$m('prices')}" publishers="{$m('publishers')}"/>     
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:stream, use map expression with computed key expressions -->
  
  <xsl:template name="m-003" use-when="true() or $RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="BOOKLIST/BOOKS/ITEM">
          <xsl:variable name="m" as="map(*)" 
                        select="map:merge(*!map{lower-case(local-name()):string(.)})"/>
          <book author="{$m('author')}" title="{$m('title')}" price="{$m('price')}" publisher="{$m('publisher')}"/>     
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:stream, use map expression to compute max and min simultaneously -->
  
  <xsl:template name="m-004" use-when="$RUN" expand-text="yes">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
          <xsl:variable name="m" as="map(*)" select="map{
          		'min': min(outermost(//PRICE)/number()),
          		'max': max(outermost(//PRICE)/number())}"/>
          <min>{$m('min')}</min> 
          <max>{$m('max')}</max>    
      </xsl:source-document>
    </out>
  </xsl:template>  
  
  <!-- within xsl:stream, use map expression selecting an empty sequence -->
  
  <xsl:template name="m-005" use-when="$RUN" expand-text="yes">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
          <xsl:variable name="m" as="map(*)" select="map{
          		'min': min(outermost(//PRICEZZZ)/number()),
          		'max': max(outermost(//PRICEZZZ)/number())}"/>
          <min>{count($m('min'))}</min> 
          <max>{count($m('max'))}</max>    
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:stream, use xsl:map with consuming key expressions -->
  
  <xsl:template name="m-006" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="BOOKLIST/BOOKS/ITEM[1]">
          <xsl:variable name="m" as="map(*)" select="map{
          		AUTHOR : true(), PUBLISHER : true(), TITLE : true()}"/>
          <xsl:if test="$m('Jane Austen') and $m('Pride and Prejudice')"><ok/></xsl:if>    
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- Streaming map expression, error due to duplicate keys -->
  
  <xsl:template name="m-007" use-when="$RUN">
    <xsl:param name="a" select="'author'"/>
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="BOOKLIST/BOOKS/ITEM">
          <xsl:variable name="m" as="map(*)" select="map{
          	'author': string(AUTHOR),
          	'title': string(TITLE),
          	'price': number(PRICE),
          	$a : string(PUBLISHER)}"/>
          <book author="{$m('author')}" title="{$m('title')}" price="{$m('price')}" publisher="{$m('publisher')}"/>     
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template> 
  
  <!-- Streaming map that contains another streaming map -->
  
  <xsl:template name="m-008" use-when="$RUN">
    <xsl:param name="a" as="item()" select="map{'outcome':'success'}"/>
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="BOOKLIST/BOOKS/ITEM">
          <xsl:variable name="m" as="map(*)">
            <xsl:map>
              <xsl:apply-templates mode="m-008" select="."/>
              <xsl:sequence select="$a"/>
            </xsl:map>
          </xsl:variable>
          <book author="{$m('author')}" title="{$m('title')}" price="{$m('price')}" publisher="{$m('publisher')}" outcome="{$m('outcome')}"/>     
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <xsl:template match="ITEM" mode="m-008" use-when="$RUN">
    <xsl:sequence 
            select="map{ 'author': string(AUTHOR),
                         'title': string(TITLE),
                         'price': number(PRICE),
                         'publisher': string(PUBLISHER)}"/>
  </xsl:template>
  
  <xsl:mode name="m-008" streamable="yes"/>
  
  <!-- Streaming map, type error because the input sequence includes a non-map -->
  
  <xsl:template name="m-009" use-when="$RUN">
    <xsl:call-template name="m-008">
      <xsl:with-param name="a" select="93.7"/>
    </xsl:call-template>
  </xsl:template>
       
  

  
</xsl:transform>  